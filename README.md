# Kafka demo using segment-io/kafka-go

Retries happen in the same topic, with a retrier daemon.

PROS:

* All messages are in the same topic, no need for multiple retry topics per topic.
* Can work for all topics, its only task would be to reinsert messages on topics.
* Re-drives can be done based on the retrier system.

CONS:

* Consumers might need logic to dismiss messages that where already processed in previous attempts.
* More complex than just cascading though retry topics.
* Need another persistence layer, since messages will have to live outside the main topic somewhere.
