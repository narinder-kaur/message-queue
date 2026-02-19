Following are the prompts used to generate the system.
This has been an incremental approach to create a system component by component.
Every single component was created, reviewed, tested and then modified with new prompt.
At many places co-pilot was not efficient to produce the expected output, there I need to add manual changes. because changes were sometime too small to be asked from a LLM to fix it.

Find the prompt here:

- I want to create an application in golang which is an amqp server. It listens to the producer requests and then save the data to an in memory messaging queue. 
message-queue should be an amqp server using this library github.com/aleybovich/carrot-mq/server


- create a service as message-queue which is an amqp server using the library github.com/aleybovich/carrot-mq/server, it listens to the producer request, accept the data, validate the data and then save the message in an in memory queue.
The producer service will read the data from a csv file, and then convert the row to a new message and send the request to amqp server to publish to consumers.
There is a consumer service, which will listen to the amqp server and when amqp server publishes the data, it will listen and save the data to a mongodb store.


- I want to create an application from scratch in golang, which can connect to producers to receive data and multiple consumer can connect to it to get data notification and data. The data format for producer and consumer should be json binary.

####
 this prompt created a code which had a consumers, which were all listening to all messages, this might be good as a broadcast, but if we need queue mechanism, it was not fullfilling this requirement.
####

- This logic should implement a concept of multiple consumers, which has a configurable behave if different consumers receives different message from broker or same.

- The producer service should be reading the data from internal/data folder. It is an csv file. the service should read this file, and there is a timestamp colume, ignore its value and replace this column value with current timestamp with same format when converting to producer messages. Csv file contains one column named labels_raw, which contains string data of key value pair separated by commas,this column should be converted to a map of key values pairs.
Convert each row into a message that message_queue application can listen and publish to consumers.

- The consumer now reads messages from the broker and stores them in MongoDB's metrics collection with the timestamp saved as a proper BSON timestamp data type.

#### this prompt saved the full message field, it should have only saved the metrics from payload. ####

- The consumer reads messages from the broker and read the payload from message and save in MongoDB's metrics collection with the timestamp saved as timestamp data type.

##### timestamps conversion in mongodb needed manual interventions ####

- create a web service as metrics. It is a rest api implemented using golang gin framework. Design and implement the following endpoints:
  - 1. List All GPUs 
    Return a list of all GPUs for which telemetry data is available. 
  2. Query Telemetry by GPU
  Return all telemetry entries for a specific GPU, ordered by time.  
     - Support optional time window filters:
        start_time (inclusive)  o end_time (inclusive)  
Example API Design:  
- GET /api/v1/gpus   
- GET /api/v1/gpus/{id}/telemetry   
- GET /api/v1/gpus/{id}/telemetry?start_time=...&end_time=... 

All endpoint should implement a sort and pagination. pagination value should be configurable. 
It should have a logger and authetication added in middleware
The service should connect to mongodb's metrics collection.


- can you convert producer code to be more production ready, where it is testable with interfaces, csv logic in separate file than main.go, and also using design principals.

- create a mock logger in internal common package, which has an interface LoggerI, and an logger object, which mimic the behave of zap.logger and can be used interchangabily in real code and test cases.

- this csv reader should be acting only as a reader, which will read the file, this struct should provide the method such as read, get header, getnextrow, has nextrow, isvalidcsv function. This file should not have a stream message logic, message streaming logic should be controlled by producer object.

- Refactor the message_queue service, it should use the logger same as used in producer. and it should also split the code in interfaces and structs which are easily testable.

- Create a test command in makefile, which will run all of the test cases in all services, create another command which will also calculate test coverage and a test coverage check command, which check if the coverage is less than 70%, it fails, else it passes.

- I want to deploy these 4 services and mongodb to a kubernetes cluster. I want that consumer and producer services should be independely scalable to max size of 10. Producer service should be deployed as a cron job which runs at a given interval. Also multiple producer job can also run in parallel.
Also deploy mongodb as a database in same cluster. create helm charts for this requirement.

- Add Service resources for message-queue and metrics (currently deployments only).
Add adjustable securityContext, probes, and more advanced HPA (memory/custom metrics).
Run a helm lint/helm template locally to validate templates

- Add a Makefile targets to build/tag/push all images.
Update Helm values.yaml to point each component to its dedicated image tag.

- charts contain same repository name in all deployments and dockerfile. Each image name should be based on its own. and tag should be a git hash of the commit. update all files in helm charts and dockerfiles.

- I see a problem in port mappings. Dockefiles does not have any port forwarding command. It should have a port open per service. Every kubernetes file should reflect the same in environement and services file, and also in value file. Please make required changes.