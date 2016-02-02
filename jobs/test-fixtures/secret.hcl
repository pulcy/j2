job "secrets" {

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
