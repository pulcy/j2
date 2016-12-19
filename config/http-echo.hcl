job "http_echo" {
	task "echo" {
		count = 1
		image = "pulcy/http-echo"
		args = ["-text", "Hello J2 World"]
		ports = [5678]
		frontend {
			domain = "hello.pulcy.com"
			port = 5678
		}
		secret "secret/dummy1" {
			environment = "DUMMY1"
		}
		secret "secret/dummy2" {
			environment = "DUMMY2"
		}
	}
}
