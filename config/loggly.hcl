// This job runs fluentd (in a docker container) on all machines to forward
// logging data to Loggly.
job "loggly" {

	task "fluentd" {
		global = true
		image = "pulcy/fluentd:latest"
		ports = ["127.0.0.1:24284:24284"]
		env {
			LOGGLY_TOKEN = "{{env "LOGGLY_TOKEN"}}"
		}
		log-driver = "none"
	}
}
