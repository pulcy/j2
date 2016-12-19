job "http_echo" {
	id = "f9fa3175-c53e-4817-b4d7-dc38d6703fe8"

	task "echo" {
		count = 1
		image = "pulcy/http-echo"
		args = ["-text", "Hello J2 World"]
		ports = [5678]
		frontend {
			domain = "hello.pulcy.com"
			port = 5678
		}
		secret "secret/base/lb/stats-user" {
			environment = "STATS_USER"
		}
	}
}
