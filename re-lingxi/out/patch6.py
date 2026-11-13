p = "extract_contracts.py"
s = open(p, encoding="utf-8").read()
s = s.replace('''    "members": ["members_action_arg"],
}''',
'''    "members": ["members_action_arg"],
    "file": ["start_line_arg", "end_line_arg"],
    "wiki_memory": ["new_content_arg", "old_text_arg"],
    "ask_user_question": ["option_label_prop", "option_description_prop"],
    "gen_image": ["aspect_ratio_arg", "input_images_arg"],
    "switch_mode": ["target_mode_arg"],
    "memory": ["edit_memory_old_not_found"],
}''')
open(p, "w", encoding="utf-8").write(s)
print("SIGN extended")
