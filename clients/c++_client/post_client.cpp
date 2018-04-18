#include <iostream>
#include <vector>
#include <sys/socket.h>
#include <arpa/inet.h>

#include "json.hpp"

using json = nlohmann::json;


int main(int argc, char **argv) {
	/*
	 * Проверка на количество параметров. 
	 * Нам нужно 2: ip и port сервера.
	 * Пример: client.exe 127.0.0.1 8080.
	 */
	if (argc != 3) {
        printf( "Error! Expected 3 parameters: ./client <ip> <port>\n");
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
        printf("Could not create socket.\n");
        return -2;
    }

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
        printf("Could not connect to remote server.\n");
        return -3;
    }

    std::string hello = "P{\"func\":\"mul\"}";
    hello.push_back('\n');
    send(socket_desc , hello.c_str() , hello.size() , 0 );
/*
    char n[1] = {'\n'};
    while (true) {
    	std::cin >> hello;
    	if (hello[0] == 'e') {
    		break;
    	}
    	send(socket_desc , hello.c_str() , hello.size() , 0 );
    	send(socket_desc , n , 1, 0 );
    }
   */ 

/*
    json s;
    s["name"] = "Habrahabr";
    std::cout << s.dump() << std::endl;
*/
    return 0;
}
