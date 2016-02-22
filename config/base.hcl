job "base" {
	id="f9fa3175-c53e-4817-b4d7-dc38d6703fe8"

	task "registrator" {
		global = true
		image = "gliderlabs/registrator:latest"
		volumes = "/var/run/docker.sock:/tmp/docker.sock"
		args = ["-ttl=120", "-ttl-refresh=90", "etcd://{{private_ipv4}}:4001/pulcy/service"]
	}

	group "load_balancer" {
		global = true

		task "certificates" {
			type = "oneshot"
			image = "pulcy/pct:0.4.1"
			volumes = "/tmp/base/lb/certs/:/certs/"
			secret "secret/base/lb/certificates-passphrase" {
                environment = "PASSPHRASE"
            }
		}

		task "lb" {
			image = "pulcy/robin:0.14.0"
			ports = ["0.0.0.0:80:80", "{{private_ipv4}}:81:81", "{{private_ipv4}}:82:82", "0.0.0.0:443:443", "0.0.0.0:7088:7088"]
			volumes = "/tmp/base/lb/certs/:/certs/"
			secret "secret/base/lb/stats-password" {
                environment = "STATS_PASSWORD"
            }
			secret "secret/base/lb/stats-user" {
                environment = "STATS_USER"
            }
			secret "secret/base/lb/acme-email" {
                environment = "ACME_EMAIL"
            }
			secret "secret/base/lb/acme-private-key" {
				file = "/acme/private-key"
            }
			secret "secret/base/lb/acme-registration" {
				file = "/acme/registration"
            }
			args = ["run",
				"--etcd-addr", "http://{{private_ipv4}}:4001/pulcy",
				"--private-key-path", "/acme/private-key",
				"--registration-path", "/acme/registration",
				"--stats-port", "7088",
				"--stats-ssl-cert", "pulcy.pem",
				"--force-ssl={{opt "force-ssl"}}",
				"--private-host", "{{private_ipv4}}",
				"--private-ssl-cert", "private.pem"
			]
		}
	}
}
