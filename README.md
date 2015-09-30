About
=====

Metronome is scheduling tool for ordering event over members in consul cluster.

When event is sent to consul cluster, Each server in the cluster will start processing parallel, But an order of processing over multiple servers is important when create system that is structured from multiple servers.
For example, all database servers should have been configured before application server, and load balancer will accept request from client after all inner servers have been ready.
Metronome can manage these order with configuration file and event queuing system on consul KVS.

![Metronome architecture](https://raw.githubusercontent.com/wiki/cloudconductor/metronome/en/diagram.png)

First, when a consul has received event, the event handler on consul trigger metronome with push subcommand, and metronome enqueue received event to event queue on consul KVS.
Metronome is polling event from event queue and dispatch it to the progress task queue with filtering condition that contains service and tag that specified by configuration file.
Each server fetch head task of progress task queue and execute operation if qualified its filter, other servers wait for finishing these operation as it fetch first task.

- [User Manual(en)](https://github.com/cloudconductor/metronome/wiki/User-Manual(en))
- [Scheduling file format(en)](https://github.com/cloudconductor/metronome/wiki/Scheduling-file-format(en))

Requirements
============

System Requirements
-------------------

- OS: Red Hat Enterprise Linux 6.5 or CentOS 6.5

Dependencies
-------------

- consul

Copyright and License
=====================

Copyright 2015 TIS inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


Contact
========

For more information: <http://cloudconductor.org/>

Report issues and requests: <https://github.com/cloudconductor/metronome/issues>

Send feedback to: <ccndctr@gmail.com>
