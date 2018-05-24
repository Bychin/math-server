#include "common.h"
#include <iostream>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>


int send_string(int socket_desc, std::string string_to_send) {
    int sended = send(socket_desc, string_to_send.c_str(), string_to_send.size(), 0);
    if (sended != string_to_send.size()) {
        std::cout << "[-] Error sending this to server:\n" << string_to_send;
        return -1;
    } else {
        return 1;
    }
}

std::string build_request(std::string msgType, json &content) {
    return msgType + content.dump() + '\n';
}