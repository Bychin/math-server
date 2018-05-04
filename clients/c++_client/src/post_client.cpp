nclude <iostream>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>

#include "json.hpp"
#include "common.h"
#define MSG_SIZE 300

using json = nlohmann::json;


int main(int argc, char **argv) {
    /*
     * Проверка на количество параметров. 
     * Нам нужно 4: ip, port сервера, логин и пароль.
     * Пример: post_client.exe 127.0.0.1 8080 arseny 123
     */
    if (argc != 5) {
        std::cout << "[-] Error! Expected 4 parameters: post_client.exe <ip> <port> <login> <password>" << std::endl;
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
    char server_message[MSG_SIZE], msgType[1];

    // Залогиниться на сервере и проверить, что все прошло удачно
    json logging_query = json({
        {"login", std::string(argv[3])},
        {"pass",  std::string(argv[4])}
    });
    send_string(socket_desc, build_request("I", logging_query)); // Послать запрос на логирование

    // Проверка ответа сервера
    if (recv(socket_desc, msgType, 1, 0) < 0) {
        std::cout << "[-] Error! Cannot read message type from server after logging" << std::endl;
        return -4;
    }

    if (recv(socket_desc, server_message, MSG_SIZE, 0) < 0) {
        std::cout << "[-] Error! Cannot read message from server after logging" << std::endl;
        return -5;
    }

    if (msgType[0] == 'E') {
        std::cout << "[-] " << server_message; 
        return -6;
    }
    std::cout << "[+] " << server_message;
    memset(server_message, 0, MSG_SIZE); // Очистить массив для дальнейшего использования

    // Послать на сервер запрос типа P на регистрацию функции.
    json post_query = json({
        {"func", "mul"}
    });
    send_string(socket_desc, build_request("P", post_query));

    // Послать на сервер информацию о том, что мы готовы заняться вычислениями.
    send_string(socket_desc, "R\n");

    // Начать общение с сервером, принимать, обрабатывать и отсылать информацию.
    while (recv(socket_desc, server_message, MSG_SIZE, 0) > 0) {
        std::cout << "[+] Task was recieved: " << server_message;

        /*
         * (Обработка запроса, вычисления).
         */

        send_string(socket_desc, "Answer for: " + std::string(server_message)); // Отправка результатов
        memset(server_message, 0, MSG_SIZE); // Очистить массив для дальнейшего использования
    }

    return 0;
}

