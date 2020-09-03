// In Node.js
//const Web3 = require('web3');

let web3 = new Web3('ws://localhost:1546');
//console.log(web3);

//web3.setProvider(new Web3.providers.WebsocketProvider('ws://localhost:1546'));

web3.eth.getAccounts().then(console.log);

web3.eth.sendTransaction({
    from: "0x63aa2b571068c4103ed1151958eea2abb9c89565",
    to: '0xabeaf76b84de7ee516daa558ec3a91fcc56221c7',
    value: "1000000000",
    type:0,
    data: ""
}, '123456').then(console.log);
