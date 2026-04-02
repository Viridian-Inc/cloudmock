#include <assert.h>
#include <stdio.h>
#include <string.h>
#include "../include/cloudmock.h"

static void test_start_stop(void) {
    printf("test_start_stop... ");
    cloudmock_options_t opts = {0, "us-east-1", "minimal"};
    cloudmock_t *cm = cloudmock_start(&opts);
    /* If cloudmock binary is not available, skip */
    if (!cm) {
        printf("SKIP (cloudmock binary not found)\n");
        return;
    }

    assert(cloudmock_port(cm) > 0);

    const char *ep = cloudmock_endpoint(cm);
    assert(ep != NULL);
    assert(strncmp(ep, "http://localhost:", 16) == 0);

    cloudmock_stop(cm);
    printf("PASS\n");
}

static void test_null_options(void) {
    printf("test_null_options... ");
    cloudmock_t *cm = cloudmock_start(NULL);
    if (!cm) {
        printf("SKIP (cloudmock binary not found)\n");
        return;
    }

    assert(cloudmock_port(cm) > 0);
    assert(cloudmock_endpoint(cm) != NULL);

    cloudmock_stop(cm);
    printf("PASS\n");
}

static void test_null_instance(void) {
    printf("test_null_instance... ");
    /* Should not crash */
    assert(cloudmock_endpoint(NULL) == NULL);
    assert(cloudmock_port(NULL) == 0);
    cloudmock_stop(NULL);
    printf("PASS\n");
}

int main(void) {
    printf("=== CloudMock C SDK Tests ===\n");
    test_null_instance();
    test_start_stop();
    test_null_options();
    printf("All tests passed.\n");
    return 0;
}
