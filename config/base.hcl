name = "base"

task "registrator" {
	global = true
	image = "gliderlabs/registrator:latest"
	volumes = "/var/run/docker.sock:/tmp/docker.sock"
	args = ["etcd://${COREOS_PRIVATE_IPV4}:4001/pulcy"]
}