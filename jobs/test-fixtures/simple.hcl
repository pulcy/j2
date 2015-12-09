job "test" {

	group "web" {
		count = 2
		task "nginx" {
			image = "alpine:3.2"
			args = ["ls", "-al", "--db", "{{link_url "test.couchdb"}}"]
			volumes-from = "storage"
			env {
				name = "ewout"
				key = "123"
				envkey = "{{env "TEST_ENV"}}"
				cattest = "{{trim (cat "file.txt")}}"
				quotetest = {{quote "hello"}}
				replacetest = "{{replace "1.2.3+git" "+git" "" -1}}"
				opttest1 = "{{opt "option1"}}"
				opttest2 = "{{opt "option2"}}"
				opttestenv = "{{opt "test-env"}}"
			}
			links = "test.couchdb"
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
		task "backup" {
			type = "oneshot"
			image = "mybackup:latest"
		}
	}

	task "dummy" {
		count = 3
		image = "alpine:latest"
	}

	task "db" {
		image = "redis:latest"
		volumes = ["/var/run/docker.sock:/tmp/docker.sock", "/etc:/etc"]
		private-frontend { }
	}

	task "global" {
		global = true
		image = "alpine:latest"
	}

	task "couchdb" {
		image = "couchdb:latest"
		private-frontend {
			port = 5984
		}
	}

	task "registrator" {
		global = true
		image = "gliderlabs/registrator:latest"
		volumes = "/var/run/docker.sock:/tmp/docker.sock"
		args = ["etcd://${COREOS_PRIVATE_IPV4}:4001/pulcy"]
		capabilities = "IPC_LOCK"
	}
}
