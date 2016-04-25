job "restartall" {

	group "lb" {
		count = 2
		restart = "all"

		task "a" {
			image = "foo-a"
		}
		task "b" {
			image = "foo-b"
			after = "a"
		}
	}
}
