CC = cc
CFLAGS = -Wall -Wextra -I include

.PHONY: all test clean

all: libcloudmock.a

libcloudmock.a: src/cloudmock.o
	ar rcs $@ $^

src/cloudmock.o: src/cloudmock.c include/cloudmock.h
	$(CC) $(CFLAGS) -c -o $@ $<

test: test/test_cloudmock
	./test/test_cloudmock

test/test_cloudmock: test/test_cloudmock.c libcloudmock.a
	$(CC) $(CFLAGS) -o $@ $< -L. -lcloudmock

clean:
	rm -f src/*.o libcloudmock.a test/test_cloudmock
