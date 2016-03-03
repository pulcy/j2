job "test" {
	id = "1234"

	constraint {
		attribute = "meta.core"
		value = "${job}"
	}

	group "web" {
		count = 2
		constraint {
			attribute = "meta.${group}"
			value = "true"
		}
		task "nginx" {
			image = "alpine:3.2"
			args = ["ls", "-al", "--db", "${link_url .db}"]
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
				jobname = "${job}"
				jobid = "${job.id}"
				groupname = "${group}"
				groupfull = "${group.full}"
				taskname = "${task}"
				taskfull = "${task.full}"
				instancename = "${instance}"
				instancefull = "${instance.full}"
				containername = "${container}"
			}
			http-check-path = "/"
			frontend {
				path-prefix = "/"
			}
			frontend {
				weight = 10
				domain = "foo.com"
			}
			frontend {
				weight = 12
				domain = "foo2.com"
				path-prefix = "/foo2"
				ssl-cert = "pulcy.pem"
				user "tester" {
					password = "foo"
				}
			}
		}
		task "storage" {
			image = "mystorage:latest"
			constraint {
				attribute = "node.id"
				value = "123456789"
			}
			link "job.redis.master" {
				type = "tcp"
				ports = [6379]
			}
		}
		task "backup" {
			type = "oneshot"
			image = "mybackup:latest"
			timer = "hourly"
		}
	}

	task "dummy" {
		count = 3
		image = "alpine:latest"
		docker-args = ["--net=host"]
	}

	task "db" {
		image = "redis:latest"
		volumes = ["/var/run/docker.sock:/tmp/docker.sock", "/etc:/etc"]
		private-frontend {
			user "admin" {
				password = "dummy"
			}
		}
	}

	task "global" {
		global = true
		image = "alpine:latest"
	}

	task "couchdb" {
		image = "couchdb:latest"
		private-frontend {
			port = 5984
			register-instance = true
		}
		volumes = "${task.volume}:/data"
	}

	task "registrator" {
		global = true
		image = "gliderlabs/registrator:latest"
		volumes = "/var/run/docker.sock:/tmp/docker.sock"
		args = ["etcd://${COREOS_PRIVATE_IPV4}:4001/pulcy"]
		capabilities = "IPC_LOCK"
	}
}
