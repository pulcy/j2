job "constraints" {

	group "group1" {
		count = 2
		task "taska" {
			image = "myserver:latest"
		}
	}

	group "group2" {
		constraint {
			// Force my tasks on a different host that contains tasks of 'g1'
			attribute = "taskgroup"
			value = "group1"
			operator = "!="
		}
		task "taskb" {
			image = "myserver:latest"
		}
	}

	group "group2global" {
		global = true
		constraint {
			// Force my tasks on a different host that contains tasks of 'g1'
			attribute = "taskgroup"
			value = "group1"
			operator = "!="
		}
		task "taskgrobalb" {
			image = "myserver:latest"
		}
	}
}
