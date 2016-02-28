// Job configuration for https://github.com/pulcy/slackcat-conf
job "slackcat_conf" {
	id="74180b83b35a41f0a28594510723eb7e"

	task "configurator" {
		global = true
		type = "oneshot"
		volumes = "/etc/:/etc/"
		image = "pulcy/slackcat-conf:latest"
		secret "secret/slackcat/webhook_url" {
			environment = "WEBHOOK_URL"
		}
	}
}
