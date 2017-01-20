job "base" {
	id="f9fa3175-c53e-4817-b4d7-dc38d6703fe8"

{{if (eq .Cluster.Orchestrator "fleet")}}

	task "registrator" {
		global = true
		image = "pulcy/registrator:0.7.2"
		volumes = "/var/run/docker.sock:/tmp/docker.sock"
		args = ["-ttl=120", "-ttl-refresh=90", "etcd://${etcd_host}:${etcd_port}/pulcy/service"]
		network = "host"
	}

	task "lbcertificates" {
		global = true
		type = "oneshot"
		image = "pulcy/pct:0.4.2"
		volumes = "/tmp/base/lb/certs/:/certs/"
		secret "secret/base/lb/certificates-passphrase" {
			environment = "PASSPHRASE"
		}
	}

	group "load_balancer" {
		global = true
		count = 2 // This splits the instances of the tasks up into 2 groups, 50% of the machines get one group, the other 50% the rest.
		restart = "all" // If one task is restarted, restart all tasks.

		task "lb" {
			image = "pulcy/robin:0.25.0"
			ports = [
				"0.0.0.0:80:80", 
				"${private_ipv4}:81:81", 
				"${private_ipv4}:82:82", 
				"0.0.0.0:443:443", 
				"0.0.0.0:7088:7088", 
				"${private_ipv4}:8055:8055",
				"${private_ipv4}:8056:8056"
			]
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
			metrics {
	            port = 8055
	            path = "/metrics"
	        }
			http-check-path = "/v1/ping"
			http-check-method = "HEAD"
			private-frontend {
				port = 8056
			}
			network = "host"
			args = ["run",
				"--etcd-endpoint", "${etcd_endpoints}",
				"--etcd-path", "/pulcy",
				"--private-key-path", "/acme/private-key",
				"--registration-path", "/acme/registration",
				"--stats-port", "7088",
				"--force-ssl={{opt "force-ssl"}}",
				"--private-host", "${private_ipv4}",
				"--private-ssl-cert", "private.pem",
				"--metrics-host", "${private_ipv4}",
				"--metrics-port", "8055"
			]
		}
	}

{{else if (eq .Cluster.Orchestrator "kubernetes")}}

	// The load-balancer exposes port 80, 443 & 7088.
	task "lb" {
		global = true
		network = "host"
		constraint {
			attribute = "meta.lb"
			value = "true"
		}
		count = 1
		image = "pulcy/robin:20170120211303"
		ports = ["0.0.0.0:80:80", "81", "82", "0.0.0.0:443:443", "0.0.0.0:7088:7088", "8055", "8056"]
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
		metrics {
			port = 8055
			path = "/metrics"
		}
		http-check-path = "/v1/ping"
		http-check-method = "HEAD"
		private-frontend {
			port = 8056
		}
		args = ["run",
			"--backend=kubernetes",
			"--log-level=debug",
			"--etcd-endpoint", "http://127.0.0.1:4001",
			"--etcd-no-sync=true",
			"--private-key-path", "/acme/private-key",
			"--registration-path", "/acme/registration",
			"--stats-port", "7088",
			"--force-ssl={{opt "force-ssl"}}",
			"--private-host", "0.0.0.0",
			"--private-ssl-cert", "private.pem",
			"--metrics-host", "0.0.0.0",
			"--metrics-port", "8055"
		]
	}


	// The dashboard exposes a Kubernetes dashboard.
	task "dashboard" {
		count = 1
		image = "gcr.io/google_containers/kubernetes-dashboard-amd64:v1.5.1"
		ports = ["0.0.0.0:30090:9090"]
		http-check-path = "/"
		frontend {
			domain = "kdb-{{opt "stack"}}.{{opt "domain"}}"
			port = 9090
			user "admin" {
				password = "foo"
			}
		}
	}

{{end}}

}
