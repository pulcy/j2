job "ceph" {

	task "monitor" {
		global = true
		image = "ceph/daemon"
		volumes = "/var/lib/ceph:/var/lib/ceph"
		args = ["mon"]
		env {
			MON_IP = "{{private_ipv4}}"
			MON_NAME = "{{hostname}}"
			KV_TYPE = "etcd"
			KV_IP = "{{private_ipv4}}"
			CEPH_PUBLIC_NETWORK = "$(echo {{private_ipv4}} | cut -d '.' -f 1,2,3 | awk '{print $1 \".0/24\"}')"
		}
		docker_args = ["--net=host"]
	}

	task "osd" {
		global = true
		image = "ceph/daemon"
		volumes = [
			"/var/lib/ceph/osd:/var/lib/ceph/osd",
			"/var/log/ceph:/var/log/ceph"
		]
		args = ["osd"]
		env {
			HOSTNAME= "{{hostname}}"
			OSD_TYPE = "directory"
			OSD_JOURNAL_SIZE = "20"
			KV_TYPE = "etcd"
			KV_IP = "{{private_ipv4}}"
		}
		docker_args = ["--net=host", "--privileged=true", "--pid=host"]
	}
}
