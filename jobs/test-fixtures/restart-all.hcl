job "restartall" {

	group "lb1" {
		global = true
		count = 1
		restart = "all"

		task "ta" {
			image = "foo-a"
		}
		task "tb" {
			image = "foo-b"
			after = "ta"
		}
	}

	group "lb2" {
		global = true
		count = 2
		restart = "all"

		task "ta" {
			image = "foo-a"
		}
		task "tb" {
			image = "foo-b"
			after = ["ta"]
		}
	}
}
