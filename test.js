// In Node.js
//const Web3 = require('web3');

if (typeof web3 !== 'undefined') {
    web3 = new Web3(web3.currentProvider);
} else {
    // set the provider you want from Web3.providers
    web3 = new Web3(new Web3.providers.WebsocketProvider("ws://localhost:1546"));
}

web3.eth.getAccounts().then(function (accounts) {
    console.log(accounts);

    web3.eth.getBalance(accounts[0]).then(function (amount) {
        console.log(amount)
    });

    console.log(web3.eth)
});

var content = 'sdfadsfasdfasdfadsfdsfasfadsfa';
console.log(web3.utils.toHex(content));

// binary
/*web3.eth.sendTransaction({
    from: "0x63aa2b571068c4103ed1151958eea2abb9c89565",
    to: '0xabeaf76b84de7ee516daa558ec3a91fcc56221c7',
    value: "1000000000",
    type:0,
    data: ""
}, '123456').then(console.log);*/


personal.unlockAccount("0x63aa2b571068c4103ed1151958eea2abb9c89565","123456",0)
personal.sendTransaction({
    from: '0x63aa2b571068c4103ed1151958eea2abb9c89565',
    type:3,
    data: '0x74657874636f6e74656e74303032'
}, '123456')
personal.sendTransaction({
    from: '0x63aa2b571068c4103ed1151958eea2abb9c89565',
    to: '0xe7916e48016bc8a2c6830e68873185af8ef3831b',
    type:5,
    data: '0x74657874636f6e74656e74303032'
}, '123456')

admin.startWS()
0xc5d88c76cb4035738631faad6ef6fc7617bb33950b93bbbf7bbf709a56928655
eth.getOrigin("textcontent002")
web3.eth.sendTransaction({
    from: '0x63aa2b571068c4103ed1151958eea2abb9c89565',
    to: '0xabeaf76b84de7ee516daa558ec3a91fcc56221c7',
    value: 0,
    type:4,
    data: web3.utils.toHex("text content")
}, '123456').then(console.log);