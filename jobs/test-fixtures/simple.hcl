name = "test"

group "web" {
	count = 2
	task "nginx" {
		image = "alpine:3.2"
		args = ["ls", "-al"]	
		volumes-from = "storage"
		env {
			name = "ewout"
			key = "123"
		}
	}	
	task "storage" {
		image = "mystorage:latest"
	}
}

task "dummy" {
	count = 3
	image = "alpine:latest"
}

task "db" {
	image = "redis:latest"
	volumes = "/var/run/docker.sock:/tmp/docker.sock"
}

task "registrator" {
	global = true
	image = "gliderlabs/registrator:latest"
	volumes = "/var/run/docker.sock:/tmp/docker.sock"
	args = ["etcd://${COREOS_PRIVATE_IPV4}:4001/pulcy"]
}