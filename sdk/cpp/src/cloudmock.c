#include "../include/cloudmock.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/wait.h>
#include <time.h>

struct cloudmock_t {
    pid_t pid;
    int port;
    char endpoint[64];
};

static int find_free_port(void) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) return -1;

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    addr.sin_port = 0;

    if (bind(sock, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
        close(sock);
        return -1;
    }

    socklen_t len = sizeof(addr);
    getsockname(sock, (struct sockaddr *)&addr, &len);
    int port = ntohs(addr.sin_port);
    close(sock);
    return port;
}

static int check_ready(int port) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) return 0;

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    addr.sin_port = htons(port);

    int result = connect(sock, (struct sockaddr *)&addr, sizeof(addr));
    close(sock);
    return result == 0;
}

cloudmock_t *cloudmock_start(const cloudmock_options_t *opts) {
    cloudmock_t *cm = calloc(1, sizeof(cloudmock_t));
    if (!cm) return NULL;

    cm->port = (opts && opts->port > 0) ? opts->port : find_free_port();
    if (cm->port < 0) { free(cm); return NULL; }

    snprintf(cm->endpoint, sizeof(cm->endpoint), "http://localhost:%d", cm->port);

    const char *profile = (opts && opts->profile) ? opts->profile : "minimal";

    char port_str[16];
    snprintf(port_str, sizeof(port_str), "%d", cm->port);

    cm->pid = fork();
    if (cm->pid < 0) { free(cm); return NULL; }

    if (cm->pid == 0) {
        /* Child process */
        setenv("CLOUDMOCK_PROFILE", profile, 1);
        setenv("CLOUDMOCK_IAM_MODE", "none", 1);
        close(STDOUT_FILENO);
        close(STDERR_FILENO);
        execlp("cloudmock", "cloudmock", "--port", port_str, NULL);
        _exit(1);
    }

    /* Wait for ready (30s timeout) */
    struct timespec ts = {0, 100000000}; /* 100ms */
    for (int i = 0; i < 300; i++) {
        if (check_ready(cm->port)) return cm;
        nanosleep(&ts, NULL);
    }

    /* Timeout — kill and cleanup */
    kill(cm->pid, SIGTERM);
    waitpid(cm->pid, NULL, 0);
    free(cm);
    return NULL;
}

const char *cloudmock_endpoint(const cloudmock_t *cm) {
    return cm ? cm->endpoint : NULL;
}

int cloudmock_port(const cloudmock_t *cm) {
    return cm ? cm->port : 0;
}

void cloudmock_stop(cloudmock_t *cm) {
    if (!cm) return;
    if (cm->pid > 0) {
        kill(cm->pid, SIGTERM);
        waitpid(cm->pid, NULL, 0);
    }
    free(cm);
}
