task "p2" {
	args = [ "--p1-url=${link_tcp .p1.redis 6379}" ]
	image = "v2:latest"
}
