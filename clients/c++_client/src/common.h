#include "json.hpp"

using json = nlohmann::json;


// Функция для посылки данных на сервер, возвращает отриц. значения в случае ошибки.
int send_string(int socket_desc, std::string string_to_send);

// Функция для преобразования типа сообщения, его наполнения к одной строке.
std::string build_request(std::string msgType, json &content);