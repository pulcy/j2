// Job configuration for https://github.com/pulcy/slackcat-conf
job "slackcat_conf" {
	id="74180b83b35a41f0a28594510723eb7e"

{{if (eq .Cluster.Orchestrator "fleet")}}

	task "configurator" {
		global = true
		type = "oneshot"
		volumes = "/etc/:/etc/"
		image = "pulcy/slackcat-conf:latest"
		secret "secret/slackcat/webhook_url" {
			environment = "WEBHOOK_URL"
		}
	}


{{else if (eq .Cluster.Orchestrator "kubernetes")}}

	task "configurator" {
		global = true
		volumes = "/etc/:/etc/"
		image = "pulcy/slackcat-conf:0.2.0"
		secret "secret/slackcat/webhook_url" {
			environment = "WEBHOOK_URL"
		}
		env {
			SLEEP = "315360000" // 10year
		}
	}

{{end}}
}
