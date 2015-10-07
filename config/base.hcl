name = "base"

task "registrator" {
	global = true
	image = "gliderlabs/registrator:latest"
	volumes = "/var/run/docker.sock:/tmp/docker.sock"
	args = ["-internal", "-ttl=120", "-ttl-refresh=90", "etcd://${COREOS_PRIVATE_IPV4}:4001/pulcy/service"]
}

task "load_balancer" {
	global = true
	image = "pulcy/lb:0.4.3"
	ports = ["0.0.0.0:80:80", "0.0.0.0:443:443"]
	args = ["--etcd-addr", "http://${COREOS_PRIVATE_IPV4}:4001/pulcy"]
}