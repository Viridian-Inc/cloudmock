package com.example;

import dev.cloudmock.CloudMock;
import org.junit.jupiter.api.*;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.boot.test.web.server.LocalServerPort;
import org.springframework.http.*;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;

import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;

@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
class AppTest {

    static CloudMock cm;

    @LocalServerPort
    int port;

    @Autowired
    TestRestTemplate rest;

    @DynamicPropertySource
    static void cloudmockProperties(DynamicPropertyRegistry registry) {
        cm = CloudMock.start();
        registry.add("AWS_ENDPOINT_URL", cm::endpoint);
    }

    @AfterAll
    static void tearDown() {
        if (cm != null) cm.stop();
    }

    private String base() {
        return "http://localhost:" + port;
    }

    @Test
    void createItem() {
        ResponseEntity<Map> res = rest.postForEntity(
            base() + "/items",
            Map.of("id", "1", "name", "Widget"),
            Map.class
        );
        assertThat(res.getStatusCode()).isEqualTo(HttpStatus.CREATED);
        assertThat(res.getBody()).containsKey("id");
    }

    @Test
    void getItem() {
        rest.postForEntity(base() + "/items", Map.of("id", "2", "name", "Gadget"), Map.class);
        ResponseEntity<Map> res = rest.getForEntity(base() + "/items/2", Map.class);
        assertThat(res.getStatusCode()).isEqualTo(HttpStatus.OK);
        assertThat(res.getBody().get("name")).isEqualTo("Gadget");
    }

    @Test
    void listItems() {
        rest.postForEntity(base() + "/items", Map.of("id", "3", "name", "A"), Map.class);
        ResponseEntity<List> res = rest.getForEntity(base() + "/items", List.class);
        assertThat(res.getStatusCode()).isEqualTo(HttpStatus.OK);
        assertThat(res.getBody()).isNotEmpty();
    }

    @Test
    void deleteItem() {
        rest.postForEntity(base() + "/items", Map.of("id", "4", "name", "Doomed"), Map.class);
        rest.delete(base() + "/items/4");
        ResponseEntity<Map> res = rest.getForEntity(base() + "/items/4", Map.class);
        assertThat(res.getStatusCode()).isEqualTo(HttpStatus.NOT_FOUND);
    }
}
