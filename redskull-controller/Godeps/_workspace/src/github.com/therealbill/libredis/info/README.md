# libredisinfo

A library to parse the strings returned by the Redis info command.


For the most part this package is used by the client package in order to
return data structures instead of raw info strings. However, you can
pass any info string to it and get those data structures out. Thus it
might be useful for parsing a string from a different client.

