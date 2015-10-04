name = "base"

task "registrator" {
	global = true
	image = "gliderlabs/registrator:latest"
	volumes = "/var/run/docker.sock:/tmp/docker.sock"
	args = ["-internal", "-ttl=120", "-ttl-refresh=90", "etcd://${COREOS_PRIVATE_IPV4}:4001/pulcy/service"]
}