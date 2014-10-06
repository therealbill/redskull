What Is Red Skull?
==================

Red Skull is a Sentinel management system. It is designed to run on each sentinel node 
you operate and provide a single, yet distributed, mechanism for managing Sentinel as 
well as interacting with it.

Overview
=========

Written in Go, Red Skull runs on each Sentinel and bootstraps itself from that 
Sentinel's configuration file. It will then interrogate any `known-sentinel` 
directives as well as run `setinel sentinels <name> for each pod found in the config file. 
It essentially crawls through your Sentinel constellation and discovers all sentinels, 
masters, and slaves.

It then provides a decent web interface for viewing and managing your sentinels, and by 
proxy the Redis instances under management. It introduces some new concepts/terminology 
and these will be explained in the documentation tree.

In addition to the front end Red Skull provides an HTTP/JSON REST-*like* interface for 
interacting with programmaticly. Adding the redis Sentinel API as another interface is 
planned as well.


Current State
=============
The initial import is of the base working code. It still likely has many bugs as it is the
result of only ~2.5 total weeks of effort and there are still much error handling to be written. 
That said, the base functionality is there and working.

The initial effort after import will be a focus on documenting Red Skull. Primarily how to 
install and use it; its design, goals, and contribution guidelines; and the direction and 
needs for it's advancement.

Can you use it for "production use". Yes. Will it destroy your setup? Not likely. It only executes read-only commands unless you click the button to make a change. 
