# Deployit

`deployit` is the pulcy service deployment tool.
It takes a job description as input and generates (fleet) unit files for all tasks in the given job.
The unit files will be pushed onto a CoreOS cluster.

## Job specification

A job is a logical group of services.

A job contains one or more task groups.
A task group is a group of tasks that are scheduled on the same machine.
A task specifies a single container.

A very basic job looks like this:

```
job "basic" {
    task {
        image = "myimage"
    }
}
```

Jobs are specified in [HCL format](https://github.com/hashicorp/hcl).
A job always has a name and only one job can be specified per file.

Objects in a job can be `task`, `group` and `constraint`.
If you add a `task` directly to a `job`, it will be automatically wrapped in a `task-group`.

The following keys can be specified on a `job`.

- `id` - `id` is used to give a job a unique identifier which is used for authentication when fetching secrets.
- `constraint` - See [Constraints](#constraints)

### Tasks

A `task` is an object that specifies something that will be executed in a container.

The following keys can be specified on a `task`.

- `args` - Contains the command line arguments passed during the start of the container.
- `environment` - Contains zero of more environment variables passed during the start of the container.
- `image` - Specifies the docker image that will be executed.
- `type` - The type of a task can be "service" (default) or "oneshot".
  Oneshot tasks are supposed to run for a while and then exit.
  Service tasks are supposed to run continuously and will be restarted in case of failure.
- `volumes-from` - Contains the name of zero or more other tasks. The volumes used by these other tasks will be
  mounted in the container of this task.
- `volumes` - Contains a list of zero or more volume mounts for the container.
  Each volumes entry must be a valid docker volume description ("hostpath:containerpath")
- `ports` - Contains a list of port specifications that specify which ports (exposed by the container) will be
  mapped into the port namespace of the machine on which the container is scheduled.
  Each port entry must be a valid docker port specification.
  Note that `ports` are not often used. In most cases you'll use a `frontend` or `private-frontend`.
- `constraint` - See [Constraints](#constraints)

```
TODO
PublicFrontEnds  []PublicFrontEnd  `json:"frontends,omitempty"`
PrivateFrontEnds []PrivateFrontEnd `json:"private-frontends,omitempty"`
HttpCheckPath    string            `json:"http-check-path,omitempty" mapstructure:"http-check-path,omitempty"`
Capabilities     []string          `json:"capabilities,omitempty"`
Links            []LinkName        `json:"links,omitempty"`
Secrets          []Secret          `json:"secrets,omitempty"`
Constraints      Constraints       `json:"constraints,omitempty"`
```

### Task groups

A `group` is an object that groups one or more tasks such that they are always scheduled on the same machine.
You use the `count` key to specify how many instances of a task-group you want to run. Each instance of a task-group
contains a container for all tasks in that task-group. Multiple instances of a task-group are not guaranteed to run on
the same machine. In fact you often want different instances to run on different machines for high availability.

The following keys can be specified on a `group`.

- `count` - Specifies how many instances of the task-group that should be created.
- `global` - If set to true, this task-group will create one instance for every machine in the cluster.
- `constraint` - See [Constraints](#constraints)

### Constraints

With constraints you can control on which machines tasks can be scheduled.
Constraints can be specified on `job`, `group` and `task` level. Constrains on a deeper level overwrite constraints
on high levels with the same `attribute`.

Here's an example of a constraint that forced a task to be scheduled on a machine that has `region=eu-west` in
its metadata.

```
constraint {
    attribute = "meta.region"
    value = "eu-west"
}
```

The following keys can be specified on a `constraint`.

- `attribute` - One of the attributes you can filter on. See [Attributes](#attributes) below.
- `value` - The value for this attribute.

#### Attributes

The following attributes can be used in constraints.

- `meta.<key>` - Refers to a key used in the metadata of a machine.
- `node.id` - Refers to the `machine-id` of a machine.
