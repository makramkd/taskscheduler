# taskscheduler

`taskscheduler` is a service that executes scheduled tasks across many servers
and collects the output of those tasks after the executions are completed.

## Architecture

The `taskscheduler` service consists of three main components:

* the server, which provides a REST API to schedule tasks and view their outputs,
* the agent, which runs on each server that the service has as an available resource to run the task, handles the scheduling,
execution, and output storage of each task,
* the database, which stores the status and output of all tasks on each server.

## Scheduler API

### Content Type

The content type of all requests and responses will be `application/json`.

### `TaskStatus`

The `TaskStatus` enumeration represents the status of a task that has been scheduled by
`taskscheduler`. The possible values for the task are the following:

* `TASK_INVALID`: the task is invalid for some reason - either the frequency provided doesn't
conform to the expected format, which is the CRON expression format.
* `TASK_RECEIVED`: the task has been successfully received and is awaiting scheduling.
* `TASK_SCHEDULED`: the task has been successfully scheduled.
* `TASK_EXECUTING`: the task is currently executing.
* `TASK_COMPLETED_SUCCESS`: the task has completed execution with no errors.
* `TASK_COMPLETED_ERROR`: the task has completed execution with errors.

### POST `/api/v1/tasks/create`

Create a scheduled, periodic task.

Request Body:
* `command` (_Required_): The command to be executed.
* `frequency` (_Required_): The frequency at which the given command is executed.

Response:
* `command_id`: A globally unique identifier representing the task that was scheduled, if the task
successfully passes verification (i.e the given command and frequency are valid).

### GET `/api/v1/tasks/{task_id}/status`

Get the status of the latest execution of the task with the given task ID.

Parameters:
* `task_id`: Path parameter. This is a globally unique identifier representing a task scheduled by `taskscheduler`.

Returns:
* `status`: The status of the task corresponding to the `TaskStatus` enumeration.
* `command`: The command for the task.
* `frequency`: The frequency at which the task is being executed.
* `latest_complete_execution_timestamp`: The timestamp of the latest complete execution of this task. In the event that the
task never completed execution, this will be `null`.
* `stdout`: If the status of the task is `TASK_COMPLETED_SUCCESS`, this will hold the data piped to STDOUT by the task. Otherwise `null`.
* `stderr`: If the status of the task is `TASK_COMPLETED_ERROR`, this will hold the data piped to STDERR by the task. Otherwise `null`.

### PUT `/api/v1/tasks/{task_id}/status`

Update the status of the given task.

Parameters:
* `task_id`: Path parameter. This is a globally unique identifier representing a task scheduled by `taskscheduler`.

Request Body:
* `server_id`: The ID of the server on which this task is being executed.
* `stdout`: The stdout of the task.
* `stderr`: The stderr of the task.
