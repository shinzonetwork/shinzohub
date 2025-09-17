// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

contract Outpost {
    //accepts payment
    //stores memory of payment
    //allows payment function to be 
    //payment(user did, policy id, payment amount)

    error PaymentAmountTooLow(uint256 amount);
    error PolicyIdDoesNotExist(string policyId);
    error DigitalIdDoesNotExist(string identity);
    error PaymentAlreadyExpired();
    error PaymentNotExpired();

    event PaymentCreated(address indexed user, string indexed policyId, uint256 paymentIndex);
    event PaymentExpired(address indexed user, uint256 indexed paymentIndex);

    struct Policy{
        string policyId;
    }

    struct DigitalID{
        address user;
        string identity; //  
    }

    struct Payment{
        Policy policy;

        uint256 amount;
        uint256 timestamp;
        uint256 expiration;
        bool expired;
    }

    mapping(address => Payment[]) public payments;
    mapping(address => DigitalID) public digitalIds;

    function payment(string memory policyId, string memory identity) public payable returns (uint256) {
        if(msg.value <= 0) revert PaymentAmountTooLow(msg.value);
        if(bytes(policyId).length == 0) revert PolicyIdDoesNotExist(policyId);
        if (bytes(identity).length == 0) revert DigitalIdDoesNotExist(identity);

        DigitalID storage digitalId = digitalIds[msg.sender];
        if (digitalId.user == address(0)) {
            digitalId.user = msg.sender;
        }
        digitalId.identity = identity;

        uint256 paymentIndex = payments[msg.sender].length;
        payments[msg.sender].push(Payment({
            policy: Policy(policyId),
            amount: msg.value,
            timestamp: block.timestamp,
            expiration: block.timestamp + 3600,
            expired: false
        }));
        emit PaymentCreated(msg.sender, policyId, paymentIndex);
        return paymentIndex;
    }

    function expirePayment(address user, uint256 paymentIndex) public returns (bool) {
        Payment storage _payment = payments[user][paymentIndex];
        if (_payment.expired) revert PaymentAlreadyExpired();
        if (block.timestamp < _payment.expiration) revert PaymentNotExpired();
        _payment.expired = true;
        emit PaymentExpired(user, paymentIndex);
        return _payment.expired;
    }   

    function getPaymentIndex(address user) public view returns (uint256) {
        return payments[user].length;
    }

    function getDigitalId(address user) public view returns (DigitalID memory) {
        return digitalIds[user];
    }

    function getPayments(address user) public view returns (Payment[] memory) {
        return payments[user];
    }
}
