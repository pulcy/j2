cluster "api" {
    id = "99132C65-0846-4479-8AA5-E500907D56AB"
    domain = "iggi.xyz"
    stack = "alpha"

    instance-count = 1

    default-options {
        api-domain = "api.iggi.xyz"
        api-ssl-cert = "api.iggi.pem"
        "force-ssl" = "true"
    }
}
