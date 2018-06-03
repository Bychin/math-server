#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include "json.hpp"

using json = nlohmann::json;

int main(int argc, char **argv) {
    json j = json::parse(argv[1]); // первым аргументом всегда является результат-json

    std::vector<int> vec = j.at("vector").get<std::vector<int>>();

    std::cout << "Result vector:\n";
    for (const auto &it : vec) {
        std::cout << it << " ";
    }
    std::cout << std::endl;
    return 0;
}