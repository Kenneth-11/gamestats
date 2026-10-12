# Build: sbt assembly -> app.jar (fat jar, sin dependencias externas en runtime)
FROM eclipse-temurin:17-jdk AS builder
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
  && curl -sL https://github.com/sbt/sbt/releases/download/v1.9.9/sbt-1.9.9.tgz | tar -xz -C /opt \
  && ln -s /opt/sbt/bin/sbt /usr/local/bin/sbt \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /build
COPY project/ ./project/
COPY build.sbt ./
RUN sbt update
COPY src/ ./src/
RUN sbt assembly

FROM eclipse-temurin:17-jre
RUN apt-get update && apt-get install -y --no-install-recommends curl \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /build/target/scala-3.3.1/app.jar app.jar
EXPOSE 8081
ENTRYPOINT ["java", "-jar", "app.jar"]
