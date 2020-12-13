# What is Zetamesh

 Zetamesh /zeta'meʃ/ is used to build a layer-three local area network on the WAN.

## Quick Start

This section will give you some instructions to make you quick start with Zetamesh.

- Build the zetamesh

    ```
    $ make
    ```

- Run a gateway at your VPS

    ```
    $ bin/zetamesh gateway
    ```

- Run the zetamesh peer node

    - Peer Node 1

        ```
        $ bin/zetamesh join --address 10.0.0.100 --gateway ${gateway}:2823
        ```

    - Peer Node 2

        ```
        $ bin/zetamesh join --address 10.0.0.101 --gateway ${gateway}:2823
        ```

- Test your LAN (at the peer node 2)

    ```
    $ ping 10.0.0.100
    PING 10.0.0.100 (10.0.0.100): 56 data bytes
    Request timeout for icmp_seq 0
    64 bytes from 10.0.0.100: icmp_seq=1 ttl=64 time=17.239 ms
    64 bytes from 10.0.0.100: icmp_seq=2 ttl=64 time=15.449 ms
    64 bytes from 10.0.0.100: icmp_seq=3 ttl=64 time=14.627 ms
    64 bytes from 10.0.0.100: icmp_seq=4 ttl=64 time=15.418 ms
    ```

## Features

- [x] Support P2P
- [x] Support relay via Gateway
- [ ] Support more operation systems
    - [x] Support MacOS
    - [x] Support Linux
    - [ ] Support Windows
    - [ ] Support iOS
    - [ ] Support Android
- [ ] Support traffic encryption

## Contribution

This project is in the early stage and many features on the roadmap. Welcome to file an issue or submit a PR.