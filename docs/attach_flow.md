# ATTACH/DETACH flow explained

NOTE, in these examples I made it so the local AMQP client starts their count for both channel and handle at `200` just to make it simpler to see who's sending what. Normally most stacks will start at 0.

## ATTACH

I send my sender link attach:
* The Channel is _my_ channel (200)
* The Handle is _my_ handle (200)
* The name is a randomly generated link name.

```json
{"Time":"2024-11-16T14:32:53.4884927-08:00","Direction":"out","Type":"*frames.PerformAttach","Header":{"Size":113,"DataOffset":2,"FrameType":0,"Channel":200,"Raw":"AAAAcQIAAMg="},"Body":{"Name":"uMx_K7TyMJ9kONWRvRaHNY7on01-7mPo-5j_n335VsFLyfh_BXr4iw","Handle":200,"Role":false,"SenderSettleMode":2,"ReceiverSettleMode":0,"Source":{"Address":"","Durable":0,"ExpiryPolicy":"session-end","Timeout":0,"Dynamic":false,"DynamicNodeProperties":null,"DistributionMode":"","Filter":null,"DefaultOutcome":null,"Outcomes":null,"Capabilities":null},"Target":{"Address":"testqueue","Durable":0,"ExpiryPolicy":"session-end","Timeout":0,"Dynamic":false,"DynamicNodeProperties":null,"Capabilities":null},"Unsettled":null,"IncompleteUnsettled":false,"InitialDeliveryCount":0,"MaxMessageSize":0,"OfferedCapabilities":null,"DesiredCapabilities":null,"Properties":null}}
```

They reply with their corresponding receiver link attach:
* Channel: 0
* Handle: 0

```json
{"Time":"2024-11-16T14:32:53.510879544-08:00","Direction":"in","Type":"*frames.PerformAttach","Header":{"Size":131,"DataOffset":2,"FrameType":0,"Channel":0,"Raw":"AAAAgwIAAAA="},"Body":{"Name":"uMx_K7TyMJ9kONWRvRaHNY7on01-7mPo-5j_n335VsFLyfh_BXr4iw","Handle":0,"Role":true,"SenderSettleMode":2,"ReceiverSettleMode":0,"Source":{"Address":"","Durable":0,"ExpiryPolicy":"session-end","Timeout":0,"Dynamic":false,"DynamicNodeProperties":null,"DistributionMode":"","Filter":null,"DefaultOutcome":null,"Outcomes":null,"Capabilities":null},"Target":{"Address":"testqueue","Durable":0,"ExpiryPolicy":"session-end","Timeout":0,"Dynamic":false,"DynamicNodeProperties":null,"Capabilities":null},"Unsettled":null,"IncompleteUnsettled":false,"InitialDeliveryCount":0,"MaxMessageSize":262144,"OfferedCapabilities":null,"DesiredCapabilities":null,"Properties":null}}
```

they reply with _their_ attach. 

*The role is the inverse of my role
* The channel is their channel
* The handle is their handle
* The name is _my_ name

To link up my local link to their remote link we join it up like this:

* our_role = !their_role        // ie: their link is always the opposite role of ours - a sender on our side requires a receiver on theirs
* link name = link name
* we should also store off their remote channel and remote handle at this point.

## DETACH

Now, when we detach:

I send them _my_ channel, _my_ handle and a closed flag.

* Channel: 200
* Handle: 200

```json
{"Time":"2024-11-16T14:32:53.604109668-08:00","Direction":"out","Type":"*frames.PerformDetach","Header":{"Size":23,"DataOffset":2,"FrameType":0,"Channel":200,"Raw":"AAAAFwIAAMg="},"Body":{"Handle":200,"Closed":true,"Error":null}}
``` 

They reply with _their_ channel, their _handle_ and a closed flag.
* Channel: 0
* Handle: 0

```json
{"Time":"2024-11-16T14:32:53.626943294-08:00","Direction":"in","Type":"*frames.PerformDetach","Header":{"Size":17,"DataOffset":2,"FrameType":0,"Channel":0,"Raw":"AAAAEQIAAAA="},"Body":{"Handle":0,"Closed":true,"Error":null}}
````

If the service initiates the DETACH it looks like this:

```json
{"Time":"2025-01-10T14:52:35.623600931-08:00","Direction":"in","Type":"*frames.PerformDetach","Connection":"PINxGfEjcf_wQTLfHmuGtXsUk0F7TWNxwsqmjXeFTLS6c9uWB5UJ0w","Header":{"Size":291,"DataOffset":2,"FrameType":0,"Channel":0,"Raw":"AAABIwIAAAA="},"Body":{"Handle":0,"Closed":true,"Error":{"Condition":"amqp:link:detach-forced","Description":"The link 'G0:32309313:Hbv6sicCOAaVyzZLq70DkrC71nnWq7TwVzHsWXOt_2GavvI3Muy1uA' is force detached. Code: publisher(link21893095). Details: Message sender was closed because entity has been deleted, updated or unexpectedly reloaded.","Info":null}}}
```

and our reply:

```json
{"Time":"2025-01-10T14:52:35.623982076-08:00","Direction":"out","Type":"*frames.PerformDetach","Connection":"PINxGfEjcf_wQTLfHmuGtXsUk0F7TWNxwsqmjXeFTLS6c9uWB5UJ0w","Header":{"Size":23,"DataOffset":2,"FrameType":0,"Channel":200,"Raw":"AAAAFwIAAMg="},"Body":{"Handle":200,"Closed":true}}
```
