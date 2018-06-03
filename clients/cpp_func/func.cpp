#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include "json.hpp"

using json = nlohmann::json;

//const std::string dataReadyPath = "./dataReady.txt";

int main(int argc, char **argv) {
    const std::string dataReadyPath = argv[1];

    json j;
    std::ifstream myfile(dataReadyPath);
    if (myfile.is_open()) {
        myfile >> j;
        myfile.close();
    }

    
    std::vector<int> vec = j.at("vector").get<std::vector<int>>();
    int scalar = j.at("scalar").get<int>();

    for (auto &i : vec) {
        i *= scalar;
    }

    std::cout << json{{"vector", vec}};
    return 0;
}