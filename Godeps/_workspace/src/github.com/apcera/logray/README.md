*Temp readme until a better one is written with examples, guidelines, etc.*

## Background

First a quick recap of some history and some current state, then into “why another logging library”.

Continuum started out with using apcera/logging, which was created early on by Brady. It is actually quite good, but usage of it was sometimes a hurdle as it could be complicated to use. You had to create log categories everywhere, often had to be passed log a context, and could log to the category, or the context with the category. It wasn’t just “call function to log, see output”. It was powerful, but lacked a simple way to use it, and the configuration was often confusing.

Some time later, it was decided to wrap apcera/logging with common/log. The goal was to make writing to a log simple, and that was true. But it also started leading to a mass propagation of log.Context everywhere. And while “call function to log, see output” was true, the code there grew to do trace logging on top of the trace logging in the logging library, and more configuration stuff on top of configuration stuff. It lead a place where there was a lot of unused code, and we used a simple base configuration that just worked and didn’t touch it.

In recent history, we’ve had a lot of complains about some of the verbosity in the logs. Some are our own logging, like not using the log levels very well or printing full Go objects everywhere, others are configuration, like in vagrant/production with runit, the date/time shows up twice (once from runit, and again from our own logs) and having trace logging on all the time. And more seriously about being able to correlate logs to actions in a process that is doing multiple things. I don’t think our internal RequestIDs, which were supposed to propagate the components, have really worked since the introduction of common/log. But is sometimes honorous to figure out “what are the log lines for this instance”.

In the meantime, the Go community start creating more logging libraries, such as logrus, glog, and others. I had first taken a look at plugging logrus into Continuum, and did get it working, but when it came to using it to solve some of our bigger problems, it didn’t fit as well.

Logray started when I was doing some work in the HM and had to look at the HM logs in vagrant to debug. I was getting rather pissed trying to find the logs for just my job. So I decided to write logray to fix this shit. 

## Goals

### Multiple outputs 

apcera/logging had this, and logrus does through hooks. This is important to start using syslog. The hooks approach is odd. In terms of practicality, could view writing to syslog as more of an output instead of hook. Additionally, it makes sense to configure it as you would an output, with things like the loglevels it should send there. I also prefered our existing output handling, which was extensible through adding outputs in code and confusing them, such as different format options. Additionally, didn’t view the formatter and output as two separate things like logrus does.

### Field propagation

Our current logging focuses on a Context, which has a name. This is often the RequestID. logrus added a fields map which allowed for more valuable metadata. I liked this and wanted to extend it, but logrus didn’t have fields building on fields. I wanted to take a context and pass it furter down into the system, and have the fields build upon it as its actions got more specific. Logrus only let you do a WithFields, which gave you an Entry, and could write multiple things on that, but couldn’t really have it turtle further down. Idea was to send a “job update” to start a job to the API Server, it gets the Job’s UUID added to its metadata, have it go to the JM with the same metadata and RequestID, have it get to the IM with the same metadata and have the InstanceUUID added, and so on.

With field propagation, it allows the “show me all logs in the IM for this instance” and “show me all the logs in the IM and HM related to this job”. This reduces the cross correlation needed before, where it was “grep IM logs for job UUID, find where it created an instance UUID, now search HM logs for the instance UUID”, and then hunt lines around it for other ones related to that message. It was broken.

### Contextually specific log formatting

At different places in the system, you care about different data related to log messages. Take the IM as an example. As basic requests come in to the system, you want to know the RequestID, such as when the JM is asking for an instance to be started, but you don’t want to see that RequestID being mentinoed 5 days later when the instance is still running. Just like as the instance is continuing to run, you likely want to see the job/instance IDs for the log line. The “show me the logs for this instance” is a common scenario we have even today, but is sometimes fragmented and a lot of guesswork.

The logging changes I’ve made allow formats/outputs to be overwritten in certain places where the format we see directly matters in a slightly different way, so a container is always setup to log the job/instance metadata, while normal component logging shows the requestID. The HM will be doing the same thing with the job/instance metadata.

Now in the IM, the logs look like this:

```
[INFO  2014-12-24 19:13:29.087473746 +0000 UTC pid=12770 requestid='' source='statemachine.go:866' job='d24f58ea-947c-4560-8d12-1824de46d5cf' instance='55f8cbbd-8552-464f-920c-5daaac0188a4'] Switching state: STARTING -> FIRST_RUNNING
```

This is incredibly long, but it also just a stopgap until we actually have syslog or something else where the metadata can be search and only display the actual message. You can actually grep this and get the specific lines you care about. Then at least the extra prefix stuff isn’t as a big a problem when all the data you have is relevant. Could probably even write an awk call to trim it. But it shows you exactly which instance it is, which job the instance is of, and even the source file/line of the line. It has other metadata that isn't displayed, like the Go package.

### Simple to use

Wanted it simple to use. So starting out, basically have to do a logger := logray.New() and can start logging. (Note, think I need to add writing to stdout by default). Want to add dields? logger.SetField(key, value). Want to pass it further down where separate fields might be set? logger2 := logger.Clone().
