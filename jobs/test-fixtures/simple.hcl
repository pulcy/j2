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
			envkey = "{{env "TESTENV"}}"
			cattest = "{{trim (cat "file.txt")}}"
			quotetest = {{quote "hello"}}
			replacetest = "{{replace "1.2.3+git" "+git" "" -1}}"
		}
		http-check-path = "/"
		frontend {
			path-prefix = "/"
		}
		frontend {
			domain = "foo.com"
		}
		frontend {
			domain = "foo2.com"
			path-prefix = "/foo2"
			ssl-cert = "pulcy.pem"
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
