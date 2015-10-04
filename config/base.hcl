name = "base"

task "registrator" {
	global = true
	image = "gliderlabs/registrator:latest"
	volumes = "/var/run/docker.sock:/tmp/docker.sock"
	args = ["-internal", "-ttl=120", "-ttl-refresh=90", "etcd://${COREOS_PRIVATE_IPV4}:4001/pulcy/service"]
}

task "load_balancer" {
	image = "arvika-ssh:5000/load-balancer:0.3.0"
	ports = ["${COREOS_PUBLIC_IPV4}:80:80", "${COREOS_PUBLIC_IPV4}:443:443"]
}