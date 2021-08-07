# Table of Contents

* [taskscheduler](#taskscheduler)
   * [Getting Started](#getting-started)
   * [Architecture](#architecture)
   * [Scheduler API](#scheduler-api)
      * [Content Type](#content-type)
      * [POST /api/v1/tasks/create](#post-apiv1taskscreate)
         * [Request Body](#request-body)
         * [Response](#response)
      * [GET /api/v1/tasks/{task_id}/latest_output](#get-apiv1taskstask_idlatest_output)
         * [Parameters](#parameters)
         * [Returns](#returns)
      * [POST /api/v1/tasks/{task_id}/complete](#post-apiv1taskstask_idcomplete)
         * [Parameters](#parameters-1)
         * [Request Body](#request-body-1)
   * [Agent API](#agent-api)
      * [POST /api/v1/tasks/schedule](#post-apiv1tasksschedule)
         * [Request Body](#request-body-2)
# taskscheduler

`taskscheduler` is a service that executes scheduled tasks across many servers
and collects the output of those tasks after the executions are completed.

## Getting Started

To get started, you can simply run the provided `docker-compose.yml`:

```
docker-compose up --build
```

This will start all the dependencies needed for the server and agent to run successfully.

Once this happens, you can run the following in another shell:

```bash
curl --request POST \
  --url http://localhost:8080/api/v1/tasks/create \
  --header 'Content-Type: application/json' \
  --data '{
	"command": "echo hello world",
	"frequency": "every 30 seconds"
}'
```

The response from the server will be a UUID representing the task you just created.

In approximately 30 seconds you can send another request to view the output of the task:

```bash
curl --request GET \
  --url http://localhost:8080/api/v1/tasks/322f0be6-d276-489f-89eb-7662c4517885/latest_output
```

Note that the UUID in the URL path will be different for you. Make sure to replace it with the ID returned by the
first `curl` command.

To test how things could be with more agents running, you can add as many as you feel like to the `docker-compose.yml` file.
Just make sure to have them listening on a different port since they're all running on the same machine, and to update
the `AVAILABLE_SERVERS` environment variable for the taskscheduler server to specify that new agent.

## Architecture

The `taskscheduler` service consists of three main components:

* the server, which provides a REST API to schedule tasks and view their outputs,
* the agent, which runs on each server that the service has as an available resource to run the task, handles the scheduling,
execution, and output storage of each task,
* the database, which stores the status and output of all tasks on each server.

## Scheduler API

### Content Type

The content type of all requests and responses will be `application/json`.

### POST `/api/v1/tasks/create`

Create a scheduled, periodic task.

#### Request Body
* `command` (*Required*): The command to be executed.
* `frequency` (*Required*): The frequency at which the given command is executed. The frequency
must be of the form `every <integer> (seconds|minutes|hours)`, otherwise the task will not be created successfully.

#### Response
* `command_id`: A globally unique identifier representing the task that was scheduled, if the task
successfully passes verification (i.e the given command and frequency are valid).

### GET `/api/v1/tasks/{task_id}/latest_output`

Get the output of the latest execution of the task with the given task ID.

#### Parameters
* `task_id`: Path parameter. This is a globally unique identifier representing a task scheduled by `taskscheduler`.

#### Returns
* `completion_time`: The timestamp of the latest complete execution of this task.
* `outputs`: This is an array of objects of the form:
```javascript
{
    "agent_id": "<some uuid>",
    "stdout": "<stdout capture>", // omitted if empty
    "stderr": "<stderr capture>"  // omitted if empty
}
```

### POST `/api/v1/tasks/{task_id}/complete`

Mark the given task as complete from a worker.

#### Parameters
* `task_id`: Path parameter. This is a globally unique identifier representing a task scheduled by `taskscheduler`.

#### Request Body
* `server_id` (*Required*): The ID of the server on which this task is being executed.
* `stdout`: The stdout of the task.
* `stderr`: The stderr of the task.

## Agent API

### POST `/api/v1/tasks/schedule`

Schedule the given task to be executed on the same host that the agent is running on.

#### Request Body
* `task_id` (*Required*): The ID of the task being scheduled.
* `command` (*Required*): The command to execute.
* `frequency` (*Required*): The frequency at which to execute the command.
