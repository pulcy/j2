job "base" {

	task "registrator" {
		global = true
		image = "gliderlabs/registrator:latest"
		volumes = "/var/run/docker.sock:/tmp/docker.sock"
		args = ["-ttl=120", "-ttl-refresh=90", "etcd://{{private_ipv4}}:4001/pulcy/service"]
	}

	group "load_balancer" {
		global = true

		task "certificates" {
			image = "pulcy/pct:0.3.1"
			env {
				PASSPHRASE = "{{opt "certificates-passphrase"}}"
			}
		}

		task "lb" {
			image = "pulcy/lb:0.10.2"
			ports = ["0.0.0.0:80:80", "{{private_ipv4}}:81:81", "0.0.0.0:443:443", "0.0.0.0:7088:7088"]
			volumes-from = "certificates"
			args = ["--etcd-addr", "http://{{private_ipv4}}:4001/pulcy",
				"--stats-port", "7088",
				"--stats-user", "admin",
				"--stats-ssl-cert", "pulcy.pem",
				"--force-ssl={{opt "force-ssl"}}",
				"--stats-password", "{{opt "stats-password"}}",
				"--private-host", "{{private_ipv4}}"
			]
		}
	}
}
