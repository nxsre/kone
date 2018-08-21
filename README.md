# KONE

The project aims to improve the experience of accessing internet in home/enterprise network.

The name "KONE" comes from [k1](https://en.wikipedia.org/wiki/Larcum_Kendall#K1), a chronometer made by Larcum Kendall and played a important role in Captain Cook's voyage.

By now, it supports:

* linux
* mac OS

## Support protocols

* tcp: http, https and socks5
* udp: only socks5 with udp support

## Use

Try finding how to use it by reading [config.example.ini](./config.example.ini)!

## Web Status

The default web status port is 6789 , just visit http://your_kone_ip:6789/ to check the kone status.

## Troubles

[ ] if the network seems down after restart kone, you can try flush your local dns cache. eg `sudo dscacheutil -flushcache;sudo killall -HUP mDNSResponder;`

## License

The MIT License (MIT) Copyright (c) 2016 xjdrew
