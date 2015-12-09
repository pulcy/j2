cluster "api" {
    domain = "iggi.xyz"
    stack = "alpha"

    instance-count = 1

    default-options {
        api-domain = "api.iggi.xyz"
        api-ssl-cert = "api.iggi.pem"
        "force-ssl" = "true"
    }
}
