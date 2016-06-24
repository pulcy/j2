job "metrics" {

	group "web" {
		count = 2
		task "server" {
			image = "myserver:latest"
			metrics {
				port = 80
			}
		}
	}

	group "custom_web" {
		task "server" {
			image = "myserver:latest"
			metrics {
				port = 90
				path = "/custom"
			}
		}
	}

	group "default_web" {
		task "server" {
			image = "myserver:latest"
			metrics {}
		}
		task "server_rules" {
			image = "myrules:latest"
			metrics {
				rules-path = "/metrics/rules"
			}
		}
	}
}
