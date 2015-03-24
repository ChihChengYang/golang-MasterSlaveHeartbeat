golang-MasterSlaveHeartbeat
================

A Go program for sending heartbeat from Slave to Master 


Usage
================

###Install

~~~
 golang-MasterSlaveHeartbeat\pkg\backuper
 golang-MasterSlaveHeartbeat\pkg\socketconn
~~~
  SlaveTableAdd("127.0.0.1:8088" , "UUID")   
  ...
  go build socketconnMaster.go
~~~
  go build socketconnSlave.go
~~~

###Examples

Step1. Run socketconnMaster waiting slaves to execute .

Step2. Run socketconnSlave, and then master arouse the heartbeat.

Step3. socketconnSlave will continue sending a signal to master.
       When socketconnSlave had a breakdown, socketconnMaster would send e-mail to inform user.


 