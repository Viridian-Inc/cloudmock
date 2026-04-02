plugins {
    kotlin("jvm") version "1.9.0"
}

group = "dev.cloudmock"
version = "1.0.0"

repositories { mavenCentral() }

dependencies {
    testImplementation("org.junit.jupiter:junit-jupiter:5.10.0")
}

tasks.test { useJUnitPlatform() }
