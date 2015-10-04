name = "test"

group "web" {
	count = 2
	task "bb2" {
		image = "alpine:3.2"
		args = ["ls", "-al"]	
		env {
			name = "ewout"
			key = "123"
		}
	}	
}

task "dummy" {
	count = 3
	image = "alpine:latest"
}