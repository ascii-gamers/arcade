small_letters = """
▄▀█ █▄▄ █▀▀ █▀▄ █▀▀ █▀▀ █▀▀ █░█ █ ░░█ █▄▀ █░░ █▀▄▀█ █▄░█ █▀█ █▀█ █▀█ █▀█ █▀ ▀█▀ █░█ █░█ █░█░█ ▀▄▀ █▄█ ▀█ 
█▀█ █▄█ █▄▄ █▄▀ ██▄ █▀░ █▄█ █▀█ █ █▄█ █░█ █▄▄ █░▀░█ █░▀█ █▄█ █▀▀ ▀▀█ █▀▄ ▄█ ░█░ █▄█ ▀▄▀ ▀▄▀▄▀ █░█ ░█░ █▄ 
"""

small_numbers = """
█▀█ ▄█ ▀█ ▀▀█ █░█ █▀ █▄▄ ▀▀█ █▀█
█▄█ ░█ █▄ ▄██ ▀▀█ ▄█ █▄█ ░░█ ▀▀█
"""

small_characters = """
█   ▀█ 
▄   ░▄ 
"""

small_letter_widths = {"I": 2, "M": 6, "N": 5, "S": 3, "W": 6, "Z": 3}
small_number_widths = {1: 3, 2: 3, 5: 3, 8: 0}
small_character_widths = [("!", 2), (" ", 2), ("?", 3)]

def gen_small_characters_map():
    res = "var smallCharacters = map[rune][]string{\n"
    width = 0

    for i in range(26):
        c = chr(65+i)

        if c not in small_letter_widths:
            small_letter_widths[c] = 4

        res += f"    '{c}': {{\n"

        for letters in small_letters.split("\n")[1:-1]:
            res += f"        \"{letters[width:width+small_letter_widths[c]-1]}\",\n"
        
        width += small_letter_widths[c]
        res += "    },\n"
    
    width = 0

    for i in range(10):
        if i not in small_number_widths:
            small_number_widths[i] = 4
        
        res += f"    '{i}': {{\n"

        for characters in small_numbers.split("\n")[1:-1]:
            res += f"        \"{characters[width:width+small_number_widths[i]-1]}\",\n"
        
        width += small_number_widths[i]
        res += "    },\n"
    
    width = 0

    for c, w in small_character_widths:
        res += f"    '{c}': {{\n"

        for characters in small_characters.split("\n")[1:-1]:
            res += f"        \"{characters[width:width+w-1]}\",\n"
        
        width += w
        res += "    },\n"
    
    res += "}"
    return res

print(gen_small_characters_map())
