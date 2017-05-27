# Folie rework

20170527
Simon Kirkby
tigger@interthingy.com

# Goal
There has been some [discussion][issue] reworking folie to handle remote nodes and streams, specifically RF nodes with a limited packet size.

To this end, a rework of folie internals is needed break them out in to structs and create a routing hub to handle and forwards packets/stream to a correct target and hooks to react to events and data. 

# Endpoints

As we are dealing with FORTH as a basis a REPL terminal should always be provided. In keeping with UNIX a terminal should be provided on STDIN,STDOUT and STDERR. Inside folie this can be routed to any available REPL (remote or local). [tve][tve] posits that there are three different endpoints that are needed to provide this.

1. Direct connection to a node with a [slip][slip] style connection to multiplex inputs
2. Remote node, connected by slip,terminal or a packet link
3. MQTT endpoint for sensor data

I think that the following endpoints would be worth adding 

4. COAP, this appears to be a popular IOT protol as well
5. Packed binary data, opaque data that gets routed
6. Logging, textual data with source information. For debugging and errors
7. Control plane, for modifing the routing and services, this should be authenticated and encrypted

# External Endpoints

Folie will need to provide a series of endpoints for interaction with devices and the network at large.

1. Direct Console , provided by running folie interactivly
2. SSH console , remote secure access to a folie instance
3. Web inteface for a GUI
4. COAP/MQTT endpoints, optional but can be activated.
5. Remote muxing, possibly across ssh

# Internal 

The internal structure for this is a departure from the current folie. The current interface is fixed.

console ---> select serial port ---> serial ( telnet or raw ) 

Reworking this into 

console ---> root console ---> switchboard ---> muxer ---> ( multiple devices )




[tve]: https://github.com/tve

[issue]: https://github.com/jeelabs/folie/issues/50
