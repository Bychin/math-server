#include <iostream>
#include <vector>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>

#include "json.hpp"
#define MSG_SIZE 300

using json = nlohmann::json;


int send_string(int socket_desc, std::string string_to_send) {
    int sended = send(socket_desc, string_to_send.c_str(), string_to_send.size(), 0);
    if (sended != string_to_send.size()) {
        std::cout << "[-] Error sending this to server:\n" << string_to_send;
        return -1;
    } else {
        return 1;
    }
}

int main(int argc, char **argv) {
	/*
	 * Проверка на количество параметров. 
	 * Нам нужно 2: ip и port сервера.
	 * Пример: client.exe 127.0.0.1 8080.
	 */
	if (argc != 3) {
        std::cout << "[-] Error! Expected 3 parameters: ./client <ip> <port>" << std::endl;
        return -1;
    }

    /*
     * Инициализация сокета. 
     * 1. AF_INET - используем стандарт IPv4.
     * 2. SOCK_STREAM - открываем TCP соединение.
     * 3. Протокол, 0 - использовать нужный протокол для данного соединения.
     * Переменная socket_desc < 0, если не удалось инициировать сокет.
     */
    int socket_desc = socket(AF_INET, SOCK_STREAM, 0);
    if (socket_desc < 0) { 
        std::cout << "[-] Could not create socket." << std::endl;
        return -2;
    }
    std::cout << "[+] Socket successfully created." << std::endl;

    /*
     * sockaddr_in - специальная структура с информацией о сервере.
     */
    struct sockaddr_in server_addr;
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = inet_addr(argv[1]);
    server_addr.sin_port = htons(atoi(argv[2]));

    /*
     * Пытаемся подключиться к серверу.
     */
    if (connect(socket_desc, (struct sockaddr *)&server_addr, sizeof(server_addr)) < 0) {
        std::cout << "[-] Could not connect to remote server." << std::endl;
        return -3;
    }
    std::cout << "[+] Successfully connected to server." << std::endl;

    /*
     * Ожидание запросов на вычисление, отправка результатов.
     * При разрыве соедининия с сервером - выход из цикла.
     */
    char server_message[MSG_SIZE];

    // Послать на сервер запрос типа P на регистрацию функции.
    std::string buf = "P{\"func\":\"mul\"}\n";
    send_string(socket_desc, buf);

    // Послать на сервер информацию о том, что мы готовы заняться
    // вычислениями.
    buf = "R\n";
    send_string(socket_desc, buf);

    // Начать общение с сервером, принимать, обрабатывать и отсылать информацию.
    while (recv(socket_desc, server_message, MSG_SIZE, 0) > 0) {
        std::cout << "[+] Task was recieved: " << server_message;

        /*
         * (Обработка запроса, вычисления).
         */

        send_string(socket_desc, "Answer for: " + std::string(server_message)); // Отправка результатов
    }

    return 0;
}
