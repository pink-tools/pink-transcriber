import os
import psutil

from pink_transcriber.config import SINGLETON_IDENTIFIERS


def _find_root_process(proc: psutil.Process, excluded_pids: list[int]) -> psutil.Process:
    root = proc
    try:
        while root.parent():
            parent = root.parent()
            if parent.pid in excluded_pids:
                break
            if parent.pid <= 1000:
                break
            root = parent
    except (psutil.NoSuchProcess, psutil.AccessDenied):
        pass
    return root


def _kill_process_tree(root: psutil.Process) -> int:
    killed = 0

    try:
        children = root.children(recursive=True)

        for child in reversed(children):
            try:
                child.kill()
                killed += 1
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                pass

        root.kill()
        killed += 1

    except (psutil.NoSuchProcess, psutil.AccessDenied):
        try:
            root.kill()
            killed += 1
        except:
            pass

    return killed


def ensure_single_instance(process_name: str) -> None:
    current_pid = os.getpid()

    parent_chain = []
    try:
        current_proc = psutil.Process(current_pid)
        while current_proc.parent():
            parent_chain.append(current_proc.parent().pid)
            current_proc = current_proc.parent()
    except (psutil.NoSuchProcess, psutil.AccessDenied):
        pass

    excluded_pids = [current_pid] + parent_chain
    killed_roots = set()

    for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
        try:
            if not proc.info['cmdline']:
                continue

            if proc.info['pid'] in excluded_pids:
                continue

            cmdline = ' '.join(proc.info['cmdline'])

            for identifier in SINGLETON_IDENTIFIERS:
                if identifier in cmdline:
                    root = _find_root_process(proc, excluded_pids)

                    if root.pid in killed_roots:
                        break

                    _kill_process_tree(root)
                    killed_roots.add(root.pid)
                    break

        except (psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
            pass
        except Exception:
            pass
