job "http_echo" {
	task "echo" {
		image = "pulcy/http-echo"
		args = ["-text", "Hello J2 World"]
		ports = [5678]
		frontend {
			domain = "hello.pulcy.com"
			port = 5678
		}
	}
}
