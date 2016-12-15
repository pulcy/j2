job "http_echo" {
	task "echo" {
		count = 2
		image = "pulcy/http-echo"
		args = ["-text", "Hello J2 World"]
		ports = [5678]
		frontend {
			domain = "hello.pulcy.com"
			port = 5678
		}
	}
}
