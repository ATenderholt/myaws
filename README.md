# Why this project?

I use Localstack at work, and while it's pretty great for running tests, I
find that it pegs the CPU pretty hard when our app is running against it for
a while. So I thought it might be fun to work on my own implementation.

And since I want to get better at Golang, seems like a good project to work
on.

# Plans

* Lambda Layers & functions
* Delegate S3 to minio
* Delegate SQS to ElasticMQ, with ability to invoke Lambda & do things with S3

