name = "test"

group "web" {
	count = 2
	task "bb" {
		image = "alpine:3.2"
		args = ["ls", "-al"]	
		env {
			name = "ewout"
			key = "123"
		}
	}	
}
