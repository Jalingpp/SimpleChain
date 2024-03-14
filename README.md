# SimpleChain
A simple implementation of blockchain systems. It can be used for secondary development.

## System Architecture
SimpleChain mainly contains four layers: nodes, blockchain, consensus, and network.
![image](https://github.com/Jalingpp/SimpleChain/assets/26080098/f30e275a-6cb9-4de8-a170-0784fc991f75)
### Nodes Layer
There are two types of nodes: client and fullnode. 
Client initiates a request, packages it into a request message, and sends it to the primary fullnode through the network layer.
Fullnode places the received client requests into a local message pool and starts an asynchronous thread for consensus.
The consensus thread packages blocks from the message pool, converts them into request messages, and hands them over to the consensus layer for sorting.
Finally, fullnodes obtain committed blocks and add them to the blockchain.
### Blockchain Layer
The blockchain layer only contains some data structures.
### Consensus Layer
The consensus algorithm adopts PBFT, which includes three stages: prepare, prepare, and commit.
### Network Layer
The network layer records the network addresses of all nodes and clients, marks the primary fullnode, and contains two algorithms: SendRequest and Broadcast.

## Other Statement
The config file records the addresses of all clients and servers, which are read and initialized by the main function.

Some test data are contained in package data, which are read and sent to the primary fullnode by clients.

The running logs of fullnodes are placed in the log file of the logout package.

Note: SimpleChain has not yet added persistence, and the storage layer only contains structures for various messages.
