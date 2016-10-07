job "secrets" {
    id = "60B6C022-5316-4F0B-AEC6-F3EE0E73A986"

    task "env_secrets" {
        image = "alpine:3.2"
        secret "secret/foo" {
            field = "somevalue" //options, defaults to "value"
            environment = "MYSECRET_KEY"
        }
        secret "secret/certificates/api.pulcy.com" {
            file = "/config/cert.pem"
        }
        args = ["--key", "${MYSECRET_KEY}"]
    }
}
