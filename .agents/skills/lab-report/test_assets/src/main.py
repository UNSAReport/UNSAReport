class Stack:
    def __init__(self):
        self.items = []

    # START-SNIPPET,stack-push
    def push(self, item):
        self.items.append(item)
    # END-SNIPPET

    # START-SNIPPET,stack-pop
    def pop(self):
        if not self.is_empty():
            return self.items.pop()
    # END-SNIPPET

    def is_empty(self):
        return len(self.items) == 0

if __name__ == "__main__":
    s = Stack()
    if s.is_empty():
        print("Stack is empty")
