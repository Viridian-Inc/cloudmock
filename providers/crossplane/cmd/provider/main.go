// Command provider is the cloudmock Crossplane provider controller-manager.
//
// It supports two modes:
//
//	cloudmock-crossplane apply -f manifest.yaml --endpoint http://localhost:4566
//	cloudmock-crossplane --endpoint http://localhost:4566  (Kubernetes reconcile mode)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	cmschema "github.com/neureaux/cloudmock/pkg/schema"
	cpinternal "github.com/neureaux/cloudmock/providers/crossplane/internal"
	"github.com/neureaux/cloudmock/services/dynamodb"
	"github.com/neureaux/cloudmock/services/ec2"
	"github.com/neureaux/cloudmock/services/s3"
	"gopkg.in/yaml.v3"
)

func main() {
	endpoint := flag.String("endpoint", envOrDefault("CLOUDMOCK_ENDPOINT", "http://localhost:4566"), "cloudmock gateway endpoint")
	region := flag.String("region", envOrDefault("AWS_REGION", "us-east-1"), "AWS region for credential scope")
	accessKey := flag.String("access-key", envOrDefault("AWS_ACCESS_KEY_ID", "test"), "AWS access key")
	secretKey := flag.String("secret-key", envOrDefault("AWS_SECRET_ACCESS_KEY", "test"), "AWS secret key")
	healthPort := flag.String("health-port", "8080", "health check port")

	flag.Parse()
	args := flag.Args()

	reg := buildRegistry()

	if len(args) >= 1 && args[0] == "apply" {
		runApply(args[1:], *endpoint, *region, *accessKey, *secretKey, reg)
		return
	}

	if len(args) >= 1 && args[0] == "generate-crds" {
		runGenerateCRDs(args[1:], reg)
		return
	}

	// Default: reconcile mode (Kubernetes controller).
	runController(*endpoint, *region, *accessKey, *secretKey, *healthPort, reg)
}

// runApply reads YAML manifests and reconciles each resource against cloudmock.
func runApply(args []string, endpoint, region, accessKey, secretKey string, reg *cmschema.Registry) {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	file := fs.String("f", "", "manifest file to apply")
	fs.Parse(args)

	if *file == "" {
		log.Fatal("usage: cloudmock-crossplane apply -f <manifest.yaml>")
	}

	data, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("reading manifest: %v", err)
	}

	// Split on YAML document separator.
	docs := splitYAMLDocs(data)

	for i, doc := range docs {
		if len(strings.TrimSpace(string(doc))) == 0 {
			continue
		}

		var manifest map[string]any
		if err := yaml.Unmarshal(doc, &manifest); err != nil {
			log.Fatalf("parsing document %d: %v", i, err)
		}

		kind, _ := manifest["kind"].(string)
		apiVersion, _ := manifest["apiVersion"].(string)

		// Skip ProviderConfig resources — they configure the provider, not create cloud resources.
		if kind == "ProviderConfig" {
			log.Printf("skipping ProviderConfig %s", metadataName(manifest))
			continue
		}

		// Find the matching schema by matching group+kind to registry entries.
		rs := findSchemaForManifest(reg, apiVersion, kind)
		if rs == nil {
			log.Printf("warning: no schema found for %s/%s, skipping", apiVersion, kind)
			continue
		}

		// Extract inputs from spec.forProvider.
		inputs := extractForProvider(manifest)
		if inputs == nil {
			log.Fatalf("document %d: spec.forProvider is missing", i)
		}

		reconciler := cpinternal.NewReconciler(endpoint, region, accessKey, secretKey, rs)

		// Try observe first to see if the resource already exists.
		importID := ""
		if rs.ImportID != "" {
			if v, ok := inputs[rs.ImportID]; ok {
				importID = fmt.Sprintf("%v", v)
			}
		}

		if importID != "" {
			exists, _, err := reconciler.Observe(importID)
			if err != nil {
				log.Fatalf("observing %s: %v", kind, err)
			}
			if exists {
				log.Printf("%s %q already exists, skipping", kind, importID)
				continue
			}
		}

		id, state, err := reconciler.Create(inputs)
		if err != nil {
			log.Fatalf("creating %s: %v", kind, err)
		}

		stateJSON, _ := json.MarshalIndent(state, "  ", "  ")
		log.Printf("created %s %q\n  state: %s", kind, id, string(stateJSON))
	}

	log.Println("apply complete")
}

// runGenerateCRDs writes CRD YAML files to the specified output directory.
func runGenerateCRDs(args []string, reg *cmschema.Registry) {
	fs := flag.NewFlagSet("generate-crds", flag.ExitOnError)
	outDir := fs.String("o", "crds", "output directory for CRD YAML files")
	fs.Parse(args)

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("creating output directory: %v", err)
	}

	crds := cpinternal.GenerateCRDs(reg)
	for _, crd := range crds {
		data, err := yaml.Marshal(crd)
		if err != nil {
			log.Fatalf("marshaling CRD: %v", err)
		}

		filename := crd.Spec.Group + "_" + crd.Spec.Names.Plural + ".yaml"
		path := filepath.Join(*outDir, filename)
		if err := os.WriteFile(path, data, 0644); err != nil {
			log.Fatalf("writing %s: %v", path, err)
		}
		log.Printf("wrote %s", path)
	}
}

// runController starts the Kubernetes controller-manager mode.
// For now this exposes a health endpoint and logs readiness.
func runController(endpoint, region, accessKey, secretKey, healthPort string, reg *cmschema.Registry) {
	log.Printf("cloudmock-crossplane provider starting")
	log.Printf("  endpoint: %s", endpoint)
	log.Printf("  region: %s", region)
	log.Printf("  resources: %d", reg.Len())

	// Create reconcilers for each resource type.
	reconcilers := map[string]*cpinternal.Reconciler{}
	for _, rs := range reg.All() {
		rsCopy := rs
		kind := cpinternal.CRDKind(rs)
		reconcilers[kind] = cpinternal.NewReconciler(endpoint, region, accessKey, secretKey, &rsCopy)
		log.Printf("  registered reconciler for %s (%s)", kind, cpinternal.CRDGroup(rs))
	}

	// Health endpoint.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := ":" + healthPort
	log.Printf("  health endpoint: %s", addr)
	log.Printf("cloudmock-crossplane provider ready")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("health server: %v", err)
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func buildRegistry() *cmschema.Registry {
	s3Svc := s3.New()
	dynamoSvc := dynamodb.New("000000000000", "us-east-1")
	ec2Svc := ec2.New("000000000000", "us-east-1")

	var tier1 []cmschema.ResourceSchema
	for _, schemas := range [][]cmschema.ResourceSchema{
		s3Svc.ResourceSchemas(),
		dynamoSvc.ResourceSchemas(),
		ec2Svc.ResourceSchemas(),
	} {
		tier1 = append(tier1, schemas...)
	}

	return cmschema.BuildRegistry(tier1, nil)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// splitYAMLDocs splits a byte slice on YAML document separators (---).
func splitYAMLDocs(data []byte) [][]byte {
	var docs [][]byte
	parts := strings.Split(string(data), "\n---")
	for _, part := range parts {
		docs = append(docs, []byte(part))
	}
	return docs
}

// metadataName extracts .metadata.name from a manifest map.
func metadataName(manifest map[string]any) string {
	if meta, ok := manifest["metadata"].(map[string]any); ok {
		if name, ok := meta["name"].(string); ok {
			return name
		}
	}
	return "<unknown>"
}

// extractForProvider extracts .spec.forProvider from a manifest map.
func extractForProvider(manifest map[string]any) map[string]any {
	spec, ok := manifest["spec"].(map[string]any)
	if !ok {
		return nil
	}
	fp, ok := spec["forProvider"].(map[string]any)
	if !ok {
		return nil
	}
	return fp
}

// findSchemaForManifest locates the registry schema matching a Crossplane manifest.
func findSchemaForManifest(reg *cmschema.Registry, apiVersion, kind string) *cmschema.ResourceSchema {
	// apiVersion is like "s3.cloudmock.io/v1alpha1"
	group := strings.Split(apiVersion, "/")[0]
	// group is like "s3.cloudmock.io" — extract service name.
	service := strings.Split(group, ".")[0]

	for _, rs := range reg.ByService(service) {
		crdKind := cpinternal.CRDKind(rs)
		if crdKind == kind {
			rsCopy := rs
			return &rsCopy
		}
	}
	return nil
}
