#ifndef CLOUDMOCK_H
#define CLOUDMOCK_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct cloudmock_t cloudmock_t;

typedef struct {
    int port;           /* 0 = auto-select */
    const char *region; /* NULL = "us-east-1" */
    const char *profile; /* NULL = "minimal" */
} cloudmock_options_t;

/* Start a CloudMock instance. Returns NULL on failure. */
cloudmock_t *cloudmock_start(const cloudmock_options_t *opts);

/* Get the endpoint URL (e.g., "http://localhost:4566"). */
const char *cloudmock_endpoint(const cloudmock_t *cm);

/* Get the port number. */
int cloudmock_port(const cloudmock_t *cm);

/* Stop and free the CloudMock instance. */
void cloudmock_stop(cloudmock_t *cm);

#ifdef __cplusplus
}
#endif

#endif /* CLOUDMOCK_H */
